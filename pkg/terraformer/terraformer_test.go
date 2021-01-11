// SPDX-FileCopyrightText: 2020 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package terraformer_test

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"syscall"

	"github.com/gardener/gardener/pkg/utils/test"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"k8s.io/apimachinery/pkg/util/clock"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/gardener/terraformer/pkg/terraformer"
	"github.com/gardener/terraformer/pkg/utils"
	testutils "github.com/gardener/terraformer/test/utils"
)

var _ = Describe("Terraformer", func() {
	Describe("#NewDefaultTerraformer", func() {
		It("should fail, if it can't create a client", func() {
			_, err := terraformer.NewDefaultTerraformer(&terraformer.Config{})
			Expect(err).To(MatchError(ContainSubstring("kubernetes client")))
		})
	})

	Describe("#Run", func() {
		var (
			fakeTerraform testutils.FakeTerraform
			tf            *terraformer.Terraformer
			baseDir       string
			paths         *terraformer.PathSet
			testObjs      *testutils.TestObjects

			resetVars func()
			logBuffer *gbytes.Buffer
		)

		BeforeEach(func() {
			var err error
			baseDir, err = ioutil.TempDir("", "tf-test-*")
			Expect(err).NotTo(HaveOccurred())

			var handle testutils.CleanupActionHandle
			handle = testutils.AddCleanupAction(func() {
				defer testutils.RemoveCleanupAction(handle)
				Expect(os.RemoveAll(baseDir)).To(Succeed())
			})

			paths = terraformer.DefaultPaths().WithBaseDir(baseDir)

			testObjs = testutils.PrepareTestObjects(ctx, testClient, "")

			logBuffer = gbytes.NewBuffer()
			multiWriter := io.MultiWriter(GinkgoWriter, logBuffer)
			resetVars = test.WithVars(
				&terraformer.Stderr, multiWriter,
			)

			tf, err = terraformer.NewTerraformer(
				&terraformer.Config{
					Namespace:                  testObjs.Namespace,
					ConfigurationConfigMapName: testObjs.ConfigurationConfigMap.Name,
					StateConfigMapName:         testObjs.StateConfigMap.Name,
					VariablesSecretName:        testObjs.VariablesSecret.Name,
					RESTConfig:                 restConfig,
				},
				zap.New(zap.UseDevMode(true), zap.WriteTo(multiWriter)),
				paths,
				clock.RealClock{},
			)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			resetVars()
		})

		Context("basic tests without terraform binary", func() {
			It("should fail, if command is not supported", func() {
				Expect(tf.Run("non-existing")).To(MatchError(ContainSubstring("not supported")))
			})
			It("should not allow to run Init directly", func() {
				Expect(tf.Run(terraformer.Init)).To(MatchError(ContainSubstring("not supported")))
			})
			It("should not allow to run Plan directly", func() {
				Expect(tf.Run(terraformer.Plan)).To(MatchError(ContainSubstring("not supported")))
			})
			It("should fail if config can't be fetched", func() {
				Expect(testClient.Delete(ctx, testObjs.ConfigurationConfigMap)).To(Succeed())
				Expect(tf.Run(terraformer.Apply)).To(MatchError(ContainSubstring("not found")))
			})
		})

		Context("successful terraform execution", func() {
			var (
				resetBinary func()
			)

			BeforeEach(func() {
				fakeTerraform = testutils.NewFakeTerraform(
					testutils.OverwriteExitCode("0"),
					testutils.OverwriteSleepDuration("50ms"),
				)

				resetBinary = test.WithVars(
					&terraformer.TerraformBinary, fakeTerraform.Path,
				)
			})

			AfterEach(func() {
				resetBinary()
			})

			It("should run Apply successfully", func() {
				Expect(tf.Run(terraformer.Apply)).To(Succeed())
				Eventually(logBuffer).Should(gbytes.Say("some terraform output"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished successfully"))
				testObjs.Refresh()
				Expect(testObjs.ConfigurationConfigMap.Finalizers).To(ContainElement(terraformer.TerraformerFinalizer))
				Expect(testObjs.StateConfigMap.Finalizers).To(ContainElement(terraformer.TerraformerFinalizer))
				Expect(testObjs.VariablesSecret.Finalizers).To(ContainElement(terraformer.TerraformerFinalizer))
			})
			It("should run Destroy successfully", func() {
				Expect(tf.Run(terraformer.Destroy)).To(Succeed())
				Eventually(logBuffer).Should(gbytes.Say("some terraform output"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished successfully"))
				testObjs.Refresh()
				Expect(testObjs.ConfigurationConfigMap.Finalizers).ToNot(ContainElement(terraformer.TerraformerFinalizer))
				Expect(testObjs.StateConfigMap.Finalizers).ToNot(ContainElement(terraformer.TerraformerFinalizer))
				Expect(testObjs.VariablesSecret.Finalizers).ToNot(ContainElement(terraformer.TerraformerFinalizer))
			})
			It("should create non-existing objects successfully on Apply", func() {
				Expect(testClient.Delete(ctx, testObjs.StateConfigMap)).To(Succeed())
				Expect(testClient.Get(ctx, testutils.ObjectKeyFromObject(testObjs.StateConfigMap), testObjs.StateConfigMap)).ToNot(Succeed())
				Expect(tf.Run(terraformer.Apply)).To(Succeed())
				Expect(testClient.Get(ctx, testutils.ObjectKeyFromObject(testObjs.StateConfigMap), testObjs.StateConfigMap)).To(Succeed())
				Expect(testObjs.StateConfigMap.Finalizers).To(ContainElement(terraformer.TerraformerFinalizer))
			})
			It("should run Validate successfully", func() {
				Expect(tf.Run(terraformer.Validate)).To(Succeed())
				Eventually(logBuffer).Should(gbytes.Say("some terraform output"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished successfully"))
			})
		})

		Context("failed terraform execution", func() {
			var (
				resetBinary        func()
				overwriteExitCodes testutils.Overwrite
			)

			BeforeEach(func() {
				overwriteExitCodes = testutils.OverwriteExitCodeForCommands(
					"init", "0",
					"apply", "42",
					"destroy", "43",
					"validate", "44",
				)
			})

			JustBeforeEach(func() {
				fakeTerraform = testutils.NewFakeTerraform(
					testutils.OverwriteSleepDuration("50ms"),
					overwriteExitCodes,
				)

				resetBinary = test.WithVars(
					&terraformer.TerraformBinary, fakeTerraform.Path,
				)
			})

			AfterEach(func() {
				resetBinary()
			})

			Context("init fails", func() {
				BeforeEach(func() {
					overwriteExitCodes = testutils.OverwriteExitCodeForCommands(
						"init", "12",
					)
				})
				It("should return exit code from terraform init", func() {
					err := tf.Run(terraformer.Apply)
					Expect(err).To(MatchError(ContainSubstring("terraform command failed")))

					var withExitCode utils.WithExitCode
					Expect(errors.As(err, &withExitCode)).To(BeTrue())
					Expect(withExitCode.ExitCode()).To(Equal(12))

					Eventually(logBuffer).Should(gbytes.Say("some terraform error"))
					Eventually(logBuffer).Should(gbytes.Say("terraform process finished with error"))
					Eventually(logBuffer).Should(gbytes.Say("triggering final state update before exiting"))
					Eventually(logBuffer).Should(gbytes.Say("successfully stored terraform state"))
				})
			})

			It("should return exit code from terraform apply", func() {
				err := tf.Run(terraformer.Apply)
				Expect(err).To(MatchError(ContainSubstring("terraform command failed")))

				var withExitCode utils.WithExitCode
				Expect(errors.As(err, &withExitCode)).To(BeTrue())
				Expect(withExitCode.ExitCode()).To(Equal(42))

				Eventually(logBuffer).Should(gbytes.Say("some terraform error"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished with error"))
				Eventually(logBuffer).Should(gbytes.Say("triggering final state update before exiting"))
				Eventually(logBuffer).Should(gbytes.Say("successfully stored terraform state"))
			})
			It("should return exit code from terraform destroy", func() {
				err := tf.Run(terraformer.Destroy)
				Expect(err).To(MatchError(ContainSubstring("terraform command failed")))

				var withExitCode utils.WithExitCode
				Expect(errors.As(err, &withExitCode)).To(BeTrue())
				Expect(withExitCode.ExitCode()).To(Equal(43))

				Eventually(logBuffer).Should(gbytes.Say("some terraform error"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished with error"))
				Eventually(logBuffer).Should(gbytes.Say("triggering final state update before exiting"))
				Eventually(logBuffer).Should(gbytes.Say("successfully stored terraform state"))
			})
			It("should return exit code from terraform validate", func() {
				err := tf.Run(terraformer.Validate)
				Expect(err).To(MatchError(ContainSubstring("terraform command failed")))

				var withExitCode utils.WithExitCode
				Expect(errors.As(err, &withExitCode)).To(BeTrue())
				Expect(withExitCode.ExitCode()).To(Equal(44))

				Eventually(logBuffer).Should(gbytes.Say("some terraform error"))
				Eventually(logBuffer).Should(gbytes.Say("terraform process finished with error"))
				Eventually(logBuffer).Should(gbytes.Say("triggering final state update before exiting"))
				Eventually(logBuffer).Should(gbytes.Say("successfully stored terraform state"))
			})

			Context("validate succeeds, but plan fails", func() {
				BeforeEach(func() {
					overwriteExitCodes = testutils.OverwriteExitCodeForCommands(
						"init", "0",
						"validate", "0",
						"plan", "45",
					)
				})
				It("should return exit code from terraform plan", func() {
					err := tf.Run(terraformer.Validate)
					Expect(err).To(MatchError(ContainSubstring("terraform command failed")))

					var withExitCode utils.WithExitCode
					Expect(errors.As(err, &withExitCode)).To(BeTrue())
					Expect(withExitCode.ExitCode()).To(Equal(45))

					Eventually(logBuffer).Should(gbytes.Say("some terraform error"))
					Eventually(logBuffer).Should(gbytes.Say("terraform process finished with error"))
					Eventually(logBuffer).Should(gbytes.Say("triggering final state update before exiting"))
					Eventually(logBuffer).Should(gbytes.Say("successfully stored terraform state"))
				})
			})
		})

		Describe("signal handling", func() {
			var (
				signalCh chan<- os.Signal

				resetVars func()
			)

			BeforeEach(func() {
				fakeTerraform = testutils.NewFakeTerraform(
					testutils.OverwriteExitCode("0"),
					testutils.OverwriteSleepDuration("200ms"),
				)

				resetVars = test.WithVars(
					&terraformer.TerraformBinary, fakeTerraform.Path,
					&terraformer.SignalNotify, func(c chan<- os.Signal, sig ...os.Signal) {
						Expect(sig).To(ConsistOf(syscall.SIGINT, syscall.SIGTERM))

						signalCh = c
					},
				)
			})

			AfterEach(func() {
				resetVars()
			})

			It("should relay SIGINT to terraform process", func(done Done) {
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					wg.Wait()
					close(done)
				}()

				go func() {
					defer GinkgoRecover()
					Expect(tf.Run(terraformer.Apply)).To(Succeed())
					wg.Done()
				}()

				Eventually(logBuffer).Should(gbytes.Say("some terraform output"), "should run terraform init")
				Eventually(logBuffer).Should(gbytes.Say("some terraform output"), "should run terraform apply")

				signalCh <- syscall.SIGINT
				Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("fake terraform received signal: %s", syscall.SIGINT.String())))

				Eventually(logBuffer).Should(gbytes.Say("terraform process finished successfully"))
				wg.Done()
			}, 1)

			It("should relay SIGTERM to terraform process", func(done Done) {
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					wg.Wait()
					close(done)
				}()

				go func() {
					defer GinkgoRecover()
					Expect(tf.Run(terraformer.Apply)).To(Succeed())
					wg.Done()
				}()

				Eventually(logBuffer).Should(gbytes.Say("some terraform output"), "should run terraform init")
				Eventually(logBuffer).Should(gbytes.Say("some terraform output"), "should run terraform apply")

				signalCh <- syscall.SIGTERM
				Eventually(logBuffer).Should(gbytes.Say(fmt.Sprintf("fake terraform received signal: %s", syscall.SIGINT.String())))

				Eventually(logBuffer).Should(gbytes.Say("terraform process finished successfully"))
				wg.Done()
			}, 1)
		})
	})
})
